#!/bin/bash
set +x
set +e

APP_TYPE='statefulset'
kubectl --selector="k8s-app=$RD_OPTION_APP_NAME" describe deployment -n $RD_OPTION_ENV_NAME 2>&1| grep "CreationTimestamp" >/dev/null 2>&1 && APP_TYPE='deployment'

case $RD_OPTION_ACTION in
"restart")
	kubectl rollout restart $APP_TYPE $RD_OPTION_APP_NAME -n $RD_OPTION_ENV_NAME
	;;
"scale")
	kubectl scale ${APP_TYPE}/${RD_OPTION_APP_NAME} --replicas=${RD_OPTION_SCALE_NUM:-1} -n $RD_OPTION_ENV_NAME
	;;
"*")
	echo 'do nothing,exit!'
	exit 1
	;;
esac

sleep 5
while :; do
	status=$(kubectl --selector="k8s-app=$RD_OPTION_APP_NAME" describe $APP_TYPE -n $RD_OPTION_ENV_NAME | grep 'Replicas:')

	if [ "$APP_TYPE" == "statefulset" ]; then
		desired=$(echo $status | awk '{print $3}')
		total=$(echo $status | awk '{print $6}')
		echo "Replicas:  ${desired} desired | ${total} total"

		if [ $desired -eq $total ]; then
			echo "Update Success !!!!!!!!!"
			exit 0
		fi
		sleep 5
	else
		desired=$(echo $status | awk '{print $2}')
		updated=$(echo $status | awk '{print $5}')
		total=$(echo $status | awk '{print $8}')
		available=$(echo $status | awk '{print $11}')
		unavailable=$(echo $status | awk '{print $14}')
		echo "Replicas:  ${desired} desired | ${updated} updated | ${total} total | ${available} available | ${unavailable} unavailable"

		if [ $desired -eq $available ] && [ $unavailable -eq 0 ]; then
			echo "Update Success !!!!!!!!!"
			exit 0
		fi
	fi
	sleep 5
done
