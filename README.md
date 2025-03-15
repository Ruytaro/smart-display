# smart-display

Go app to display statistics on a usb-serial graphic lcd

## Motivation

I wanted to have a display to monitor linux boxes but I can't have a standard display because space constrictions. So I went to Ali-X and look for cheap lcd displays and found the "turing smart display" that is just what I want. But it's software is Win only and anyway i didn't want to run a closed software from a unkonwn source on my machines.
Looking at the internet, there are some great open source re-implemenations of the software (like <https://github.com/mathoudebine/turing-smart-screen-python>). My issue is that i wanted to monitor also some OpenWRT routers, and didn't want to fight againt the serpents. So I decided to make a small project in Go that solves my issue.

## Usage

- ```-tty ttyUSB0``` sets the output tty to use to ```/dev/ttyUSB0``` (by default is ttyACM0)
- ```-test``` tests the display with default paterns
- ```-d``` enables "debug mode" in which writes to a png instead
- ```-path / -path /tmp/``` monitors the space usage on the desired paths (up to 5 paths)
- ```-rate 2``` sets the time in seconds between updates (by default is 5s)
