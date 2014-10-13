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

# discover local executables [adb, vlc]
%w{adb vlc}.each do |e|
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

execs = %w{mimic screenrecord}
execs.each do |f|
  # clean any already existing executables
  usb_sh "rm /data/local/tmp/#{f}"
  # copy them
  `adb -d push bin/#{f} /data/local/tmp/#{f}`
  # mark as executable
  usb_sh "chmod 700 /data/local/tmp/#{f}"
end

# enable adb over tcpip
`adb tcpip 5555`
sleep 1
# connect over tcpip
`adb connect #{ip}:5555`

# sleep a bit
sleep 1

# start mimic over tcpip
thr = Thread.new { puts `adb -s '#{ip}:5555' shell '/data/local/tmp/screenrecord --bit-rate 12000000 --time-limit 1800 --raw stdout | /data/local/tmp/mimic'` }

trap ("INT") do
  thr.kill
  `adb disconnect #{ip}:5555`
end

# sleep a bit more...
sleep 3

puts "============================================"
puts "Screen mirroring running. CTRL-C to quit."
puts "You can disconnect your device from USB now."
puts "============================================"
puts ""
puts "Here's some debug output from VLC:"

# start VLC
`vlc rtsp://#{ip}:5554`
# all done.
