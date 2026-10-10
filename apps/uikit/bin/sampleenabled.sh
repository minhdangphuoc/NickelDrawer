#!/bin/sh
# A switch: stores true or false so the next screen read shows it, and fills the readout.
. "$(dirname "$0")/lib.sh"
remember_value sampleenabled toggle
