package cli

import "fmt"

func cmdVideo(socket string, args cliVideoCmd) int {
	if args.Zoom < 1 {
		return fail(fmt.Errorf("--zoom must be >= 1"))
	}
	if err := RunVideoPreview(args.Host, args.Port, args.RefreshMS, args.Zoom, socket); err != nil {
		return fail(err)
	}
	return 0
}
