#!/usr/bin/bash

virtualenv venv
source venv/bin/activate
pip3 install -r requirements.txt
deactivate