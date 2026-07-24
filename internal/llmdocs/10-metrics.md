# Metrics and Core Web Vitals thresholds

| key | name | Good | Poor | unit | notes |
| --- | --- | --- | --- | --- | --- |
| `lcp` | Largest Contentful Paint | ≤2500 | >4000 | ms | Core Web Vital |
| `inp` | Interaction to Next Paint | ≤200 | >500 | ms | Core Web Vital |
| `cls` | Cumulative Layout Shift | ≤0.1 | >0.25 | ratio | Core Web Vital (unitless) |
| `fcp` | First Contentful Paint | ≤1800 | >3000 | ms | |
| `ttfb` | Time to First Byte | ≤600 | >1200 | ms | the API metric is "experimental" |
| `rtt` | Round Trip Time | ≤75 | >275 | ms | |
| `ol` | Onload | — | — | ms | BigQuery only |
| `fid` | First Input Delay | ≤100 | >300 | ms | legacy; BigQuery only |

"Good%" is the fraction of experiences in the good bucket (the first, fastest
bucket).

`cls` is a unitless ratio. Every other metric is in milliseconds — do not label
a `cls` value with "ms".

## API metric short key → CrUX API metric name

For cross-referencing the raw API:

| short key | CrUX API name |
| --- | --- |
| `lcp` | `largest_contentful_paint` |
| `cls` | `cumulative_layout_shift` |
| `inp` | `interaction_to_next_paint` |
| `fcp` | `first_contentful_paint` |
| `ttfb` | `experimental_time_to_first_byte` |
| `rtt` | `round_trip_time` |
