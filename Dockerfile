FROM ubuntu:latest

RUN apt-get -y update
RUN apt-get -y upgrade
RUN apt-get -y install traceroute iputils-ping mtr-tiny
ADD ./trace /trace
RUN /trace