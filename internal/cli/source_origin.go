package cli

import "context"

func (g gitSource) Origin(ctx context.Context, path string) (string, bool, error) {
	return g.client.Origin(ctx, path)
}
