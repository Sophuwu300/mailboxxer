#!/bin/python3.11

import email
import sys
from email import policy
from email.parser import BytesParser
import os


def openeml(file):
    with open(file, 'rb') as fp:
        eml = BytesParser(policy=policy.default).parse(fp)
    return eml


def showparts(eml):
    for part in eml.walk():
        if part.get_payload(decode=True) is not None:
            #print(part.get_payload(decode=True).decode('utf-8', errors='ignore'))
            print(part.get_content_type())


for file in sys.argv[1:]:
    eml = openeml(file)
    print(file)
    showparts(eml)
    print()
