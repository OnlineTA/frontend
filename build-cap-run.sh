#!/bin/sh
go build
sudo setcap cap_chown=ep frontend
./frontend
