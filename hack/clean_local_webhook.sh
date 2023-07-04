#!/bin/bash
set -ex

oc delete validatingwebhookconfiguration/vopenstackdataplane.kb.io --ignore-not-found
oc delete mutatingwebhookconfiguration/mopenstackdataplane.kb.io --ignore-not-found
