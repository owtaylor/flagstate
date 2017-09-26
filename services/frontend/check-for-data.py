#!/usr/bin/python3

import requests
import sys

repositories = requests.get(sys.argv[1] + '/v2/_catalog').json()['repositories']
if len(repositories) > 0:
    sys.exit(0)
else:
    sys.exit(1)
