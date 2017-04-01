#!/usr/bin/env python
# If you've already got an export of your text messages in CSV format (as
# created by the SMStoText app), you can use this import script to shove them
# all into the database in one go. Again, this was also thrown together - do not
# expect this to be fast or awesome.
# Change path in the open() statement below to your CSV file and the URL to your
# installation.

URL = "https://whoever:whatever@wherever.co.uk"

import csv
import requests
import time
import hashlib
import base64

with open('import.csv', 'r') as fh:
    smsreader = csv.reader(fh)
    for row in smsreader:
        if row[0] == 'Date':
            continue

        date = time.strftime('%d-%M-%Y', time.strptime(row[0], "%Y-%M-%d"))
        timestamp = time.strftime('%H.%M', time.strptime(row[1], "%H:%M:%S"))
        hashts = time.strftime('%H:%M:00', time.strptime(row[1], "%H:%M:%S"))
        f = row[4]
        message = row[5]

        h = hashlib.sha256()
        h.update("{} {}".format(row[0], hashts))
        h.update(f)
        h.update(message)

        resp = requests.get(URL+"/sms?hash={}".format(h.hexdigest()))
        if resp.status_code == 404:
            payload = {
                "time": timestamp,
                "date": date,
                "from": f,
                "message": base64.b64encode(message),
            }
            resp = requests.post(URL+"/sms", payload)
            print "{}: {}".format(resp.status_code, message)
        else:
            print "Duplicate message, not sent: {}".format(message)
