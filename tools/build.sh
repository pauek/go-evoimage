#!/bin/bash

(for file in *.go; do
   echo go build $file
done) | parallel
