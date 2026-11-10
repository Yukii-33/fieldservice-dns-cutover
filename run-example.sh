#!/bin/sh
set -eu

go run . -order work-order.example.json -mode apply
