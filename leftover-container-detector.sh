#!/bin/bash
if [ -f "services.yaml" ]; then
    rm services.yaml
fi

 # download services.yaml
 wget --quiet https://raw.githubusercontent.com/Financial-Times/up-service-files/master/services.yaml
[ -z "$FLEETCTL_TUNNEL" ] && echo "Need to set FLEETCTL_TUNNEL" && exit 1;
echo "Environment: $FLEETCTL_TUNNEL"
# get running container information
containers=$(for id in `fleetctl list-machines | awk '{print $1}' | tail -5 | tr -d "." | xargs`; do \
                echo `fleetctl ssh $id docker ps | awk '{print $2}' | grep -v "^ID"`; \
            done | xargs | tr ' ' '\n' | sort | uniq -c | sort)

# send containers info to go app to check for duplicates
./leftover-container-detector "$containers"
