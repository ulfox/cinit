# ----- ------ ------
FROM golang:1.17.1-bullseye as base


ARG RELEASE_DATE
ARG VERSION

ENV RELEASE_DATE=${RELEASE_DATE:-"2021-10-02"}
LABEL release-date="${RELEASE_DATE}"

RUN mkdir -vp /opt
COPY . /opt/
WORKDIR /opt

RUN make deps


RUN make cinitd VERSION="${VERSION}" 
RUN make cli VERSION="${VERSION}" 

RUN install "/opt/cinit-daemon" /usr/bin/cinit-daemon
RUN install /opt/cinit /usr/bin/cinit

RUN rm "/opt/cinit-daemon" /opt/cinit

RUN mkdir -vp /data
VOLUME [ "/data" ]
WORKDIR /data

ENTRYPOINT [ "cinit-daemon" ]
