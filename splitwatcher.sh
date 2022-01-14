#!/bin/sh

# This script automates tmux to create a two-row layout, where the top row is
# evenly split among several "file watchers" (tail -f instances), and the
# bottom row is an interactive terminal.
#
# It is automatically set up so that 'exit' will close the entire session.
# Also, closing any pane will exit the session too.
#
# Influential environment variables:
#
# * FORCE_PS1 - override the PS1 of the shell(s) in the tmux session.
# * STARTDIR - set the initial CWD for the main interactive shell.
# * INJECT_COMMANDS - inject additional commands to be run in the terminal on
#   startup
# * WELCOME - if defined, this file will be displayed in the interactive
#   terminal.
#
# This guide[0] is a helpful resource for understanding what is going on here.
#
# 0 - https://www.arp242.net/tmux.html

set -e
set -u

# To avoid having a sidecar file, we pack our config with us. However, tmux
# does not like reading it's config from file descriptors, so we can't simply
# use process substitution or pipe it to standard in. Therefore we create a
# temp file, and create a trap to clean it up when we exit.
TMUX_CONFIG_FILE="$(mktemp)"
cat > "$TMUX_CONFIG_FILE" << EOF
set -g pane-border-format "#P: #{pane_title}"
set -g pane-border-status top
set -g status off
set-option -g destroy-unattached off
set-option -g remain-on-exit off
set -g mouse on
EOF
trap "rm -f '$TMUX_CONFIG_FILE'" EXIT HUP INT QUIT PIPE TERM

tmux -f "$TMUX_CONFIG_FILE" new-session -d -s monitor
tmux new-window -d -t "=monitor" -n "overview"

# Close the initial window.
tmux send-keys -t "+0" "exit" Enter

# Create the initial split between the file watchers on the top and the
# interactive terminal on the bottom.
tmux split-window -v -t "=monitor:=overview"

# This should be run in all terminals before anything else.
PREFACE="alias exit='tmux kill-session -t =monitor'; clear"

# If we set a PS1 in the parent process, force it to be passed through via
# PREFACE.
set +u
if [ ! -z "$FORCE_PS1" ] ; then
	PREFACE="export PS1='$FORCE_PS1'; $PREFACE"
fi
if [ ! -z "$INJECT_COMMANDS" ] ; then
	PREFACE="$INJECT_COMMANDS; $PREFACE"
fi
if [ -z "$WELCOME" ] ; then
	WELCOME=/dev/null
fi
set -u

# Set up the first file watcher. Observe that we set it up so that if the tail
# is killed, the session is killed too.
tmux send-keys -t "=monitor:=overview.0" "$PREFACE ; echo -en '\033]0;monitor for $1\a'; clear" Enter
tmux send-keys -t "=monitor:=overview.0" "sleep 1 ; clear ; tail -f '$1'" Enter
shift

# Initially, we have:
#
#   +-----+
#   | .0  |
#   +-----+
#   | .1  |
#   +-----+
#
# After the first loop iteration, we have:
#
#   +-----+------+
#   | .0  |  .1  |
#   +-----+------+
#   |    .2      |
#   +-----+------+
#
# And so on. We need to track this (using N) because we want to run "tail" in
# the second to last pane number. After this loop is over, we need to know N so
# that we know which pane to make the interactive terminal.
N=1
while [ $# -gt 0 ] ; do
	# Create a new split for the additional files to watch.
	tmux split-window -h -t "=monitor:=overview.0"

	# And launch tail inside of that terminal...
	tmux send-keys -t "=monitor:=overview.$N" "$PREFACE ; echo -en '\033]0;monitor for $1\a'; clear" Enter
	tmux send-keys -t "=monitor:=overview.$N" "sleep 1 ; clear ; tail -f '$1'" Enter
	shift

	N=$(expr $N + 1)
done

# .0 is the top-left pane, select-layout -E splits all panes next to the
# current pane out evenly. This keeps the interactive pane on the bottom, but
# splits the top file watcher panes.
tmux select-layout -t "=monitor:=overview.0" -E

# Set up a "watcher" window that simply checks if any pane has been closed, and
# exits the session if so. Mind the spaces in the strings here, they are
# needed!
tmux new-window -d -t "=monitor" -n "exitwatcher"
tmux send-keys -t "=monitor:=exitwatcher.0" "$PREFACE; N=$N"
tmux send-keys -t "=monitor:=exitwatcher.0" '; while true ; do if [ "$(tmux list-panes -t =monitor:=overview | cut -d: -f1 | wc -l)" -ne '
# recall N is 0-indexed
tmux send-keys -t "=monitor:=exitwatcher.0" "$(expr $N + 1)"
tmux send-keys -t "=monitor:=exitwatcher.0" ' ] ; then tmux kill-session -t =monitor ; fi ; sleep 1 ; done'
tmux send-keys -t "=monitor:=exitwatcher.0" Enter

# Set up the interactive shell with a helpful title.
tmux select-window -t "=monitor:=overview"
tmux select-pane -t "=monitor:=overview.$N"
set +u
if [ ! -z "$STARTDIR" ] ; then
	tmux send-keys -t "=monitor:=overview.$N" "cd '$STARTDIR'" Enter
fi
set -u
tmux send-keys -t "=monitor:=overview.$N" "$PREFACE; echo -en '\033]0;interactive shell, \"exit\" or closing any pane will close the tmux session\a'; clear" Enter
tmux send-keys -t "=monitor:=overview.$N" "sleep 0.2 ; history -c ; clear ; cat '$WELCOME'" Enter

tmux attach-session -t "=monitor"
