FROM gcr.io/distroless/base:debug
COPY ["nsswitch.conf", "/etc/nsswitch.conf"]
COPY opm /bin/opm
ENTRYPOINT ["/bin/opm"]
