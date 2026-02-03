# Cost Estimation

Estimated monthly costs are approximate and depend on traffic, log volume, and region.

## Dev
- NAT Gateways: moderate
- WAF: low (logging disabled)
- CloudTrail/GuardDuty: optional

## Staging
- NAT Gateways: moderate
- WAF: moderate
- CloudTrail/GuardDuty: enabled

## Production
- NAT Gateways: higher (multi-AZ)
- WAF: moderate/high (logging enabled)
- CloudTrail/GuardDuty: enabled

## Cost Optimization
- Disable WAF logging in dev
- Reduce retention periods in non-prod
- Use scheduled shutdown of non-prod resources
