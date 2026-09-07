package execx

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

func detachedIdentity(pid int) (string, bool, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if detachedProcEntryGone(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	// comm is parenthesized and may itself contain spaces or parentheses.
	end := strings.LastIndexByte(string(data), ')')
	if end < 0 {
		return "", false, errors.New("malformed process stat")
	}
	fields := strings.Fields(string(data[end+1:]))
	if len(fields) < 20 {
		return "", false, errors.New("truncated process stat")
	}
	if fields[0] == "Z" || fields[0] == "X" {
		return "", false, nil
	}
	if _, err := strconv.ParseUint(fields[19], 10, 64); err != nil {
		return "", false, fmt.Errorf("invalid process start time: %w", err)
	}
	boot, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		return "", false, err
	}
	if strings.TrimSpace(string(boot)) == "" {
		return "", false, errors.New("missing kernel boot identity")
	}
	return strings.TrimSpace(string(boot)) + ":" + fields[19], true, nil
}

// procfs can open a PID's stat entry successfully, then return ESRCH from its
// read handler after the task exits. Both outcomes mean this entry vanished;
// callers must still distinguish leader absence from an unstable group census.
func detachedProcEntryGone(err error) bool {
	return errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ESRCH)
}

func detachedGroupAlive(pgid int) (bool, error) {
	if err := syscall.Kill(-pgid, 0); errors.Is(err, syscall.ESRCH) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	var prior string
	for pass := 0; pass < 2; pass++ {
		alive, zombies, err := detachedGroupCensus(pgid)
		if err != nil || alive {
			return alive, err
		}
		if pass > 0 && zombies != prior {
			return false, errors.New("process group membership changed during termination observation")
		}
		prior = zombies
	}
	return false, nil
}

func detachedGroupCensus(pgid int) (bool, string, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return false, "", err
	}
	zombies := []string{}
	unstable := false
	for _, entry := range entries {
		if _, err := strconv.Atoi(entry.Name()); err != nil {
			continue
		}
		data, err := os.ReadFile("/proc/" + entry.Name() + "/stat")
		if detachedProcEntryGone(err) {
			unstable = true
			continue
		}
		if err != nil {
			return false, "", err
		}
		end := strings.LastIndexByte(string(data), ')')
		if end < 0 {
			return false, "", errors.New("malformed process group stat")
		}
		fields := strings.Fields(string(data[end+1:]))
		if len(fields) < 4 {
			return false, "", errors.New("truncated process group stat")
		}
		group, err := strconv.Atoi(fields[2])
		if err != nil {
			return false, "", err
		}
		if group == pgid && fields[0] != "Z" && fields[0] != "X" {
			return true, "", nil
		}
		if group == pgid {
			zombies = append(zombies, entry.Name())
		}
	}
	// /proc is not an atomic group snapshot. Refuse a census containing vanished
	// processes, and require the same zombie membership in two full snapshots:
	// this prevents a rapidly exiting parent from hiding a newly born child.
	if unstable {
		return false, "", errors.New("process census changed during termination observation")
	}
	sort.Strings(zombies)
	return false, strings.Join(zombies, ","), nil
}
