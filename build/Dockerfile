FROM scratch

ARG TARGETARCH

COPY ca-certificates.crt /etc/ssl/certs/
COPY notify-${TARGETARCH} /notify

EXPOSE 8000

ENTRYPOINT ["/notify"]
