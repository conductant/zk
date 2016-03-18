# Zookeeper

# Pull base image.
FROM ubuntu:14.04

ADD build/linux-amd64/zk /usr/local/bin/zk

RUN apt-get update
RUN apt-get install -y software-properties-common git-core wget

# Java
RUN apt-get install -y --no-install-recommends openjdk-7-jdk
RUN java -version

# Zookeeper
ADD install/zookeeper-3.4.6 /usr/local/zookeeper-3.4.6
RUN cd /usr/local && ln -s zookeeper-3.4.6 zookeeper

# Exhibitor
ADD install/exhibitor-1.5.1 /usr/local/exhibitor-1.5.1
RUN cd /usr/local && ln -s exhibitor-1.5.1 exhibitor

# Expose volumes.  This allows us to use the image as data containers as well
VOLUME /usr/local/zookeeper/conf
VOLUME /var/log/zookeeper
VOLUME /var/zookeeper

WORKDIR /var/zookeeper

EXPOSE 2181 2888 3888 8080
ENTRYPOINT ["zk"]