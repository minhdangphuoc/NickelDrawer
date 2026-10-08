# LocalSend app

Receives books from a phone via LocalSend into `/mnt/onboard/LocalSend`.

`bin/localgo` is not committed (third-party binary). `make dist` copies it from
`build/localgo-linux-armv7` (built from `third_party/localgo`) into `bin/localgo`.

State lives in `data/` (PIN in `data/pin.txt`, config in `data/home`). While running, localgo logs to `/tmp/localsend.log` so nothing stays open on the USB-visible storage; "Stop receiving" appends it to `data/log.txt` (last 500 lines kept).
