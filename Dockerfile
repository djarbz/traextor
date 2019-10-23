FROM gcr.io/distroless/base
COPY bin/traextor /
ENTRYPOINT ["/traextor"]
