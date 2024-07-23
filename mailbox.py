#!/bin/python3.11

import email
from email import policy
from email.parser import BytesParser
import os


def openeml(file):
    with open(file, 'rb') as fp:
        eml = BytesParser(policy=policy.default).parse(fp)
    return eml


def showparts(eml):
    for part in eml.walk():
        if part.get_content_disposition() == None and part.get_content_maintype() == 'text':
            print(part.get_content_maintype(), part.get_content_subtype(), len(part.get_payload(decode=True)))
        elif part.get_content_disposition() == 'attachment':
            print(part.get_content_disposition(),part.get_content_maintype(), part.get_content_subtype(), len(part.get_payload(decode=True)))


eml = openeml('test/pro.eml')
print(eml.get('Subject'), eml.get('From'), eml.get('Date'))
showparts(eml)