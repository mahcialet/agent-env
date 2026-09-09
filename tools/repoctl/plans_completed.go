package main

import "fmt"

// Completion does not waive declared dependencies. Use the consumer's proven
// delivery, not a mutable retained branch, when checking stacked ancestry.
func planCompletedDependencies(root string, g *planGraph, p planMetadata) error {
	base, err := planBaseRevision(root, p.BaseBranch)
	if err != nil {
		return err
	}
	if err := planMergeProof(root, p, base); err != nil {
		return fmt.Errorf("%s completion: %w", p.PlanID, err)
	}
	for _, d := range p.DependsOn {
		dep, ok := g.ByID[d.PlanID]
		if !ok {
			return fmt.Errorf("%s: unknown dependency %s", p.PlanID, d.PlanID)
		}
		if d.Satisfaction != "stacked" {
			if dep.Status != "completed" {
				return fmt.Errorf("%s: merged dependency %s is not completed", p.PlanID, dep.PlanID)
			}
			if err := planMergeProof(root, dep, base); err != nil {
				return fmt.Errorf("%s: merged dependency %s: %w", p.PlanID, dep.PlanID, err)
			}
			continue
		}
		delivered, err := planStackedHead(root, p)
		if err != nil {
			return err
		}
		tip, err := planStackedHead(root, dep)
		if err != nil {
			return fmt.Errorf("%s: stacked dependency %s: %w", p.PlanID, dep.PlanID, err)
		}
		if !planAncestor(root, tip, delivered) {
			return fmt.Errorf("%s: stacked dependency %s is not inherited by completed delivery", p.PlanID, dep.PlanID)
		}
		present, err := planIdentityAt(root, tip, dep.PlanID)
		if err != nil || !present {
			return fmt.Errorf("%s: stacked dependency %s lacks proven Plan identity", p.PlanID, dep.PlanID)
		}
		if dep.Status == "active" {
			if err := planMetadataMatchesRevision(root, tip, dep); err != nil {
				return fmt.Errorf("%s: stacked dependency %s: %w", p.PlanID, dep.PlanID, err)
			}
			if err := planCommitProvenance(root, dep, tip, g, map[string]bool{}, false); err != nil {
				return fmt.Errorf("%s: stacked dependency %s: %w", p.PlanID, dep.PlanID, err)
			}
		}
	}
	return nil
}
