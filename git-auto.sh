#!/bin/bash

echo "What's your commit message? (type: description)"
read commit_message
git add .
git commit -m "$commit_message"
gig push