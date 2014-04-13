#!/usr/bin/env ruby

require 'ipaddr'
require 'socket'

# something that looks like an ip addr
PATTERN_IP = /\d{1,3}(\.\d{1,3}){3}/

# check if an executable exists on host
def cmd?(name)
  system "command -v #{name} >/dev/null 2>&1"
end

# run a shell command via adb/usb
def usb_sh(cmd)
  cmd = "adb -d shell #{cmd}"
  out = `#{cmd}`
  die "`#{cmd}` failed" unless $?.to_i == 0
  out.strip
end

# print a message, then quit
def die(msg)
  puts msg
  exit 1
end

# discover local executables [adb, nc, mplayer]
%w{adb nc mplayer}.each do |e|
  die "#{e} executable not found on PATH" unless cmd? e
end

# check device OS version
api = usb_sh('getprop ro.build.version.sdk').to_i
die "Device api #{api} < 19, at least 4.4/19 needed" if api < 19

# check that device is arm
unless usb_sh('getprop ro.product.cpu.abi2') == 'armeabi'
  die "Device is not arm"
end

# discover device IP
intf = usb_sh 'getprop wifi.interface'
ip = usb_sh "getprop dhcp.#{intf}.ipaddress"
# fall back to ifconfig if device is not using dhcp
if /^#{PATTERN_IP}$/.match(ip)
  mask_addr = usb_sh("getprop dhcp.#{intf}.mask")
else
  # try to get it through ifconfig
  if /ip\s+(#{PATTERN_IP})\s.*mask\s+(#{PATTERN_IP})\s/.match usb_sh("ifconfig #{intf}")
    ip = $1
    mask_addr = $2
  else
    die "cannot determine device ip"
  end
end

# look for host IP in the same subnet
# get the CIDR suffix, count the bits
mask = IPAddr.new(mask_addr).to_i.to_s(2).count("1")
net = IPAddr.new("#{ip}/#{mask}")
host_ip = Socket.ip_address_list.detect { |intf|
  intf.ipv4? && !intf.ipv4_loopback? && !intf.ipv4_multicast? &&
    net.include?(IPAddr.new(intf.ip_address))
}.ip_address

die "Could not detect host ip. Are you on the same wifi as the device?" unless host_ip

#TODO print some nice info?

# clean the pipe
usb_sh 'rm /data/local/tmp/mimic_host'
# copy the binaries
%w{mimic_arm nc_arm mkfifo_arm}.each do |f|
  `adb -d push bin/#{f} /data/local/tmp/#{f}`
  usb_sh "chmod 755 /data/local/tmp/#{f}"
end
usb_sh '/data/local/tmp/mkfifo_arm /data/local/tmp/mimic_host'
usb_sh 'chmod 666 /data/local/tmp/mimic_host'

# enable adb over tcpip
`adb tcpip 5555`
# connect over tcpip
`adb connect #{ip}:5555`
# start local nc listening
`rm mimic_device`
`mkfifo mimic_device`
thrs = []
thrs << Thread.new { `nc -l -p 58247 > mimic_device` }
sleep 3
thrs << Thread.new { `mplayer -demuxer h264es -fps 60 mimic_device` }
thrs << Thread.new { puts `adb -s "#{ip}:5555" shell "/data/local/tmp/nc_arm #{host_ip} 58247 < /data/local/tmp/mimic_host"` }
sleep 2

trap ("INT") do
  thrs.each { |thr| thr.kill }
  `adb disconnect #{ip}:5555`
end

puts "Screen mirroring running. CTRL-C to quit."
puts "You can disconnect your device from USB now."

# start mimic
puts `adb -s "#{ip}:5555" shell /data/local/tmp/mimic_arm --bit-rate 4000000 --time-limit 1800 --raw /data/local/tmp/mimic_host`
# all done.
