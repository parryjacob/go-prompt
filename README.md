# go-prompt
A simple but nice looking PS1 prompt for Bash

## Usage
To use this, you should run `make` to build it and then add the following to your `~/.bashrc`:
```
export PS1='$(/path/to/prompt $?)'
```

This also supports passing a 2nd parameter that will be recognized as the time required to execute the previous command. Getting bash to pass this information is more complicated, bu the following snippet in your `~/.bashrc` should accomplish that:
```
function cmd_timer_start() {
    cmd_timer=${cmd_timer:-$SECONDS}
}
function cmd_timer_stop() {
    cmd_timer_show=$((SECONDS - $cmd_timer))
    unset cmd_timer
}
trap 'cmd_timer_start' DEBUG

function set_prompt() {
    LAST_CMD=$?
    cmd_timer_stop
    TIMER=$cmd_timer_show
    PS1='$(/path/to/prompt $LAST_CMD $TIMER)'
}
PROMPT_COMMAND=set_prompt
```