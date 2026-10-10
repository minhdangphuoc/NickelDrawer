#!/bin/sh
# Starts a fake job of three steps, five seconds apart, for the Steps page. The job writes
# data/job as "<label key>|<done>|<total>|<detail>"; state.sh reads it. Nothing waits for it.
. "$(dirname "$0")/lib.sh"

JOB_STEP_SECONDS=5
JOB_TOTAL=3

job_is_running() {
    [ -f "$DATA_DIR/job.pid" ] && kill -0 "$(cat "$DATA_DIR/job.pid")" 2>/dev/null
}

# The job itself, run in the background by start_job.
run_job() {
    step=1
    while [ "$step" -le "$JOB_TOTAL" ]; do
        write_file job "msg.steps_downloading|$step|$JOB_TOTAL|sample-$step.epub"
        sleep "$JOB_STEP_SECONDS"
        step=$((step + 1))
    done
    write_file job "msg.steps_done||"
    rm -f "$DATA_DIR/job.pid"
}

start_job() {
    mkdir -p "$DATA_DIR" || fail_with "$APP_DIR" "cannot create $DATA_DIR"
    write_file job "msg.steps_downloading|0|$JOB_TOTAL|"
    start_detached "$0" --run
    echo_toast started
}

if [ "${1:-}" = --run ]; then
    printf '%s\n' "$$" >"$DATA_DIR/job.pid"
    run_job
elif job_is_running; then
    echo_toast busy
else
    start_job
fi
