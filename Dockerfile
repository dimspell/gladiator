FROM gcr.io/distroless/static

# Add build-time metadata
ARG BUILD_DATE
ARG VERSION
ARG GIT_COMMIT

LABEL org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${GIT_COMMIT}"

# Copy the compiled binary
COPY ./gladiator /gladiator

# Document the ports that will be exposed
EXPOSE 2137
EXPOSE 9999

ENTRYPOINT ["/gladiator"]