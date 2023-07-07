# Build go project
FROM golang:1.20 as go-build

# Add everything
COPY . /usr/src/multi-networkpolicy-tc

WORKDIR /usr/src/multi-networkpolicy-tc
RUN go build ./cmd/multi-networkpolicy-tc/

# Build iproute2
FROM quay.io/centos/centos:stream8 as iproute-build

ARG IPROUTE2_TAG=v5.17.0

RUN dnf -q -y groupinstall "Development Tools" && dnf -q -y install libmnl-devel-1.0.4-6.el8 && dnf clean all

RUN git clone --branch ${IPROUTE2_TAG} https://github.com/shemminger/iproute2.git

WORKDIR /iproute2
RUN make && make install


# collect everything into target container
# TODO(adrianc): once we switch to native netlink to drive TC  we can switch back to distroless
#FROM gcr.io/distroless/base
FROM quay.io/centos/centos:stream8
LABEL io.k8s.display-name="Multus NetworkPolicy TC" \
      io.k8s.description="This is a component provides NetworkPolicy objects for secondary interfaces created with Multus CNI"
RUN dnf -q -y install libmnl-1.0.4-6.el8 && dnf clean all
COPY --from=go-build /usr/src/multi-networkpolicy-tc/multi-networkpolicy-tc /usr/bin
COPY --from=iproute-build /sbin/tc /sbin
WORKDIR /usr/bin

ENTRYPOINT ["multi-networkpolicy-tc"]
