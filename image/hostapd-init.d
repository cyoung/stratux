#!/bin/sh

#TODO: change DAEMON_SBIN
#TODO: add hostapd-edimax binary to image [http://www.daveconroy.com/wp3/wp-content/uploads/2013/07/hostapd.zip]

### BEGIN INIT INFO
# Provides:		hostapd
# Required-Start:	$remote_fs
# Required-Stop:	$remote_fs
# Should-Start:		$network
# Should-Stop:
# Default-Start:	2 3 4 5
# Default-Stop:		0 1 6
# Short-Description:	Advanced IEEE 802.11 management daemon
# Description:		Userspace IEEE 802.11 AP and IEEE 802.1X/WPA/WPA2/EAP
#			Authenticator
### END INIT INFO

# Detect RPi version.
#  Per http://elinux.org/RPi_HardwareHistory

DAEMON_CONF=/etc/hostapd/hostapd.conf
DAEMON_SBIN=/usr/sbin/hostapd
RPI_REV=`cat /proc/cpuinfo | grep 'Revision' | awk '{print $3}' | sed 's/^1000//'`
if [ "$RPI_REV" -eq "a01041" ] || [ "$RPI_REV" -eq "a21041" ] ; then
 # This is a RPi2B.
 DAEMON_CONF=/etc/hostapd/hostapd-edimax.conf
 DAEMON_SBIN=/usr/sbin/hostapd-edimax
fi
if [ "$RPI_REV" -eq "a01041" ] ; then
 # This is a RPi3B.
 DAEMON_CONF=/etc/hostapd/hostapd.conf
fi


PATH=/sbin:/bin:/usr/sbin:/usr/bin
DAEMON_DEFS=/etc/default/hostapd
NAME=hostapd
DESC="advanced IEEE 802.11 management"
PIDFILE=/run/hostapd.pid

[ -x "$DAEMON_SBIN" ] || exit 0
[ -s "$DAEMON_DEFS" ] && . /etc/default/hostapd
[ -n "$DAEMON_CONF" ] || exit 0

DAEMON_OPTS="-B -P $PIDFILE $DAEMON_OPTS $DAEMON_CONF"

. /lib/lsb/init-functions

case "$1" in
  start)
	log_daemon_msg "Starting $DESC" "$NAME"
	start-stop-daemon --start --oknodo --quiet --exec "$DAEMON_SBIN" \
		--pidfile "$PIDFILE" -- $DAEMON_OPTS >/dev/null
	log_end_msg "$?"
	;;
  stop)
	log_daemon_msg "Stopping $DESC" "$NAME"
	start-stop-daemon --stop --oknodo --quiet --exec "$DAEMON_SBIN" \
		--pidfile "$PIDFILE"
	log_end_msg "$?"
	;;
  reload)
  	log_daemon_msg "Reloading $DESC" "$NAME"
	start-stop-daemon --stop --signal HUP --exec "$DAEMON_SBIN" \
		--pidfile "$PIDFILE"
	log_end_msg "$?"
	;;
  restart|force-reload)
  	$0 stop
	sleep 8
	$0 start
	;;
  status)
	status_of_proc "$DAEMON_SBIN" "$NAME"
	exit $?
	;;
  *)
	N=/etc/init.d/$NAME
	echo "Usage: $N {start|stop|restart|force-reload|reload|status}" >&2
	exit 1
	;;
esac

exit 0
