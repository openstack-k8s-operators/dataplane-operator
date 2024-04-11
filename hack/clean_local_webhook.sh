#!/bin/bash
set -ex

oc delete validatingwebhookconfiguration/vopenstackdataplanenodeset.kb.io --ignore-not-found
oc delete mutatingwebhookconfiguration/mopenstackdataplanenodeset.kb.io --ignore-not-found
oc delete validatingwebhookconfiguration/vopenstackdataplanedeployment.kb.io --ignore-not-found
oc delete mutatingwebhookconfiguration/mopenstackdataplanedeployment.kb.io --ignore-not-found
oc delete validatingwebhookconfiguration/vopenstackdataplaneservice.kb.io --ignore-not-found
oc delete mutatingwebhookconfiguration/mopenstackdataplaneservice.kb.io --ignore-not-found
