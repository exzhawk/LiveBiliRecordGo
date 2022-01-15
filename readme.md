# LiveBiliRecordGo
Just another Bilibili Live recorder

## Usage
To use this script, you need two IDs: `urlId` and `roomId`. see below.

After getting theme, replace IDs in `ws.go` correspondingly.

Run the script with `go run ws.go` to start monitoring and recording.

The script do terminate when encountering errors. Use `while 1` or systemd or supervisor, etc., to keep it working.

### to get `urlId`

digits in url is `urlId`.

e.g. `urlId` for https://live.bilibili.com/281 is `281`.

### to get `roomId`

visit https://api.live.bilibili.com/room/v1/Room/room_init?id={urlId} to get roomId.

e.g. visit https://api.live.bilibili.com/room/v1/Room/room_init?id=281

and there is a string in it like `"room_id":49728`. `49728` is the `roomId`.


## FAQ

Q: Why do I have to get the `roomId` manually? You even have a `getRoomId` function, and it has been used!

A: Script changes from time to time. I'm too lazy to not use a global variable. Please make a pull request.
