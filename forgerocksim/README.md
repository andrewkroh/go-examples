# ForgeRock API simulator

Simulates responses from `https://<tenant-env-fqdn>/monitoring/logs/sources`.

- https://backstage.forgerock.com/docs/idcloud/latest/tenants/audit-debug-logs.html
- https://backstage.forgerock.com/knowledge/kb/article/a37739488

# Notes

## Test Cases

1. Request logs from last 30 days.
2. Paginate when a response include pageResultCookie.
3. Resume from end timestamp of last request upon a restart.
4. Recover appropriately from HTTP 50x errors.

## Subtleties of the API

1. Largest allowable time window in a request is exactly 24 hours. In order to
request the last 30 days then we need to make 30 separate requests for 24h
periods.
