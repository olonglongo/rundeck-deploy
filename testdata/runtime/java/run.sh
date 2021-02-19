#!/bin/bash

export JAVA_OPTS

# Run main program
if [[ "$ENV" == "prd" ]];then
    exec java $JAVA_OPTS \
            -XX:+UseG1GC \
            -XX:+UseCompressedOops \
            -XX:+UseCompressedClassPointers \
            -XX:AutoBoxCacheMax=20000 \
            -XX:+AlwaysPreTouch \
            -jar /app/blackbox-server.jar
else
    exec java $JAVA_OPTS \
            -XX:+UseG1GC \
            -XX:+UseCompressedOops \
            -XX:+UseCompressedClassPointers \
            -XX:AutoBoxCacheMax=20000 \
            -XX:+AlwaysPreTouch \
            -jar /app/blackbox-server.jar
fi

#if [[ "$ENV" == "prd" ]];then
#    exec java $JAVA_OPTS \
#	    -XX:+UseG1GC -XX:MaxGCPauseMillis=50 \
#            -XX:-OmitStackTraceInFastThrow \
#	    -Dcom.sun.management.jmxremote \
#	    -Dcom.sun.management.jmxremote.port=10080 -Dcom.sun.management.jmxremote.ssl=false \
#	    -Dcom.sun.management.jmxremote.authenticate=false \
#	    -Djava.security.egd=file:/dev/./urandom -jar /app/blackbox-server.jar
#else
#    exec java $JAVA_OPTS \
#	    -XX:+UseG1GC -XX:MaxGCPauseMillis=200 \
#            -XX:-OmitStackTraceInFastThrow \
#	    -Dcom.sun.management.jmxremote \
#	    -Dcom.sun.management.jmxremote.port=10080 -Dcom.sun.management.jmxremote.ssl=false \
#	    -Dcom.sun.management.jmxremote.authenticate=false \
#	    -Djava.security.egd=file:/dev/./urandom -jar /app/blackbox-server.jar
#fi
#
