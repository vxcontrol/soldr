#!/bin/bash

# It's fixing trouble "librt.so: cannot open shared object file: No such file or directory"
ln -s /lib/x86_64-linux-gnu/librt.so.1 /lib/x86_64-linux-gnu/librt.so

# It's fixing trouble "libpthread.so: cannot open shared object file: No such file or directory"
ln -s /lib/x86_64-linux-gnu/libpthread.so.0 /lib/x86_64-linux-gnu/libpthread.so
