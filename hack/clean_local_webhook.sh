#!/bin/bash
set -ex

oc delete validatingwebhookconfiguration/vopenstackdataplanenodeset.kb.io --ignore-not-found
oc delete mutatingwebhookconfiguration/mopenstackdataplanenodeset.kb.io --ignore-not-found
