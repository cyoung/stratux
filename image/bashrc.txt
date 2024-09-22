# ~/.bashrc: executed by bash(1) for non-login shells.

# Note: PS1 and umask are already set in /etc/profile. You should not
# need this unless you want different defaults for root.
# PS1='${debian_chroot:+($debian_chroot)}\h:\w\$ '
# umask 022

# You may uncomment the following lines if you want `ls' to be colorized:
# export LS_OPTIONS='--color=auto'
# eval "`dircolors`"
# alias ls='ls $LS_OPTIONS'
# alias ll='ls $LS_OPTIONS -l'
# alias l='ls $LS_OPTIONS -lA'
#
# Some more alias to avoid making mistakes:
# alias rm='rm -i'
# alias cp='cp -i'
# alias mv='mv -i'
export PATH=/root/go/bin:/opt/stratux/bin:${PATH}
export GOROOT=/root/go
export GOPATH=/root/go_path


# This will allow users to keep an alias file after this file is added
if [ -f /root/.aliases ]; then
. /root/.aliases
fi

# Useful aliases for stratux debugging
if [ -f /root/.stxAliases ]; then
. /root/.stxAliases
fi
