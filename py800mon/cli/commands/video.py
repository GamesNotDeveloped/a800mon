from ...video import run_video_preview


def register(subparsers):
    video = subparsers.add_parser(
        "video", help="Preview UDP video stream in a Tk window."
    )
    video.add_argument(
        "--host", default="127.0.0.1", help="Host/IP to bind for UDP stream."
    )
    video.add_argument(
        "--port", type=int, default=6502, help="UDP port to bind."
    )
    video.add_argument(
        "--refresh-ms",
        type=int,
        default=33,
        help="UI refresh interval in milliseconds.",
    )
    video.add_argument(
        "--zoom",
        type=int,
        default=1,
        help="Initial integer zoom factor for the window.",
    )
    video.set_defaults(func=cmd_video)


def cmd_video(args):
    if args.zoom < 1:
        raise SystemExit("--zoom must be >= 1")
    run_video_preview(
        args.host,
        args.port,
        args.refresh_ms,
        zoom=args.zoom,
        socket_path=args.socket,
    )
    return 0
