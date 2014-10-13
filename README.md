Mimic
=====

Mimic is a tiny proof-of-concept of doing android device screen mirroring to a
desktop over WiFi. It consists of two separate ARM binaries: a modified AOSP
_screenrecord_ binary that spews out raw h264 on `STDOUT` and a small RTSP/RTP
server written in Go that broadcasts raw h264 from `STDIN`.

**Disclaimer: this is a huge hack. Don't expect it to work reliably, if at
all.**

Running
-------

### The Fast Way

You're using either Linux or OS X, you have vlc and ruby and you have a single
device attached over adb where said device is on the same LAN as your computer:

```bash
ruby mimic.rb
```

This script will reconnect adb over tcp and start mimic (so you can untether
once mirroring works).

### The Not-So-Fast Manual Way

The other way is to manually retrace the steps that the ruby scripts takes:

```bash
# installing binaries
cd mimic/checkout/dir
adb -d push bin/screenrecord /data/local/tmp/screenrecord
adb -d push bin/mimic /data/local/tmp/mimic
adb -d chmod 700 /data/local/tmp/screenrecord
adb -d chmod 700 /data/local/tmp/mimic

# connecting adb over tcpip
DEVICE_IP=`adb shell getprop dhcp.wlan0.ipaddress` # if this fails you'll need
                                                   # to get the IP in some
                                                   # other way

# start adb in tcpip mode on the device
adb -d tcpip 5555
# connect
adb connect $DEVICE_IP:5555

# ok to disconnect usb now, lets start mimic
BITRATE=8000000
TIMELIMIT=1800
adb -s "$DEVICE_IP:5555" shell "/data/local/tmp/screenrecord \
  --bit-rate $BITRATE --time-limit $TIMELIMIT --raw stdout   \
  | /data/local/tmp/mimic"

# once done, disconnect the tcpip adb
adb disconnect $DEVICE_IP:5555
```

Building from source
--------------------

To build from source you'll need to check out AOSP, apply the patch in
`diff/screenrecord/frameworks_av.diff` and compile screenrecord from source:

```bash
cd aosp/checkout/dir
pushd frameworks/av
cat mimic/checkout/dir/diff/screenrecord/frameworks_av.diff | patch
popd
make screenrecord
cp out/target/product/generic/system/bin/screenrecord mimic/checkout/dir/bin/
```

To build the RTSP/RTP server you'll need to bootstrap go for Linux/ARM and then
build:

```bash
cd $GOROOT/src
GOOS=linux GOARCH=arm ./make.bash
cd mimic/checkout/dir
GOOS=linux GOARCH=arm go build -o bin/mimic
```

Plans
-----

Currently there's a slight latency in the order a few seconds (hence the name,
it looks like the desktop mirror is "mimicing" the device).

There's also a few packet / NALU too large warnings in VLC.

Plans are to look into both of these issues.

License
-------

Standard Apache 2.0. See the license file.
