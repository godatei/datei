# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/cc-debian12:nonroot
WORKDIR /
COPY dist/datei /datei
COPY dist/*.spdx.json /usr/local/share/sbom/
USER 65532:65532
ENTRYPOINT ["/datei"]
CMD ["serve"]
