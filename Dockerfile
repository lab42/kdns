# Setup
FROM alpine:3.20 AS setup
RUN addgroup --gid 10000 -S appgroup && \
    adduser --uid 10000 -S appuser -G appgroup

FROM scratch
COPY --from=setup /etc/passwd /etc/passwd
COPY kdns /kdns
USER appuser
EXPOSE 5353
ENTRYPOINT ["/kdns"]
