# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static-debian12:nonroot@sha256:a9329520abc449e3b14d5bc3a6ffae065bdde0f02667fa10880c49b35c109fd1
WORKDIR /
COPY dist/datei /datei
COPY dist/*.spdx.json /usr/local/share/sbom/
USER 65532:65532
ENTRYPOINT ["/datei"]
CMD ["serve"]
