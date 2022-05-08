import json, sys, os
x = json.load(sys.stdin)

# Reducers %v

try:
    print(json.dumps(x))
except:
    print(json.dumps(list(x)))
