#!/bin/bash

pid=$(pidof main)
kill $pid
