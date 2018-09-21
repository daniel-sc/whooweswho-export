# whooweswho-export

Export expense sheet from www.whooweswho.net to CSV.

For private sheets you'd need to copy the cookie from the browser to authenticate (use `-headers`).

```
Usage of whooweswho-export:
  -headers string
        additional request headers, e.g. "Cookie:session_cookie123,X-My-Header:42"
  -names string
        names to replace IDs, e.g. "123456->Arnold,987654->Schwarz"
  -output string
        csv output file (default "expenses.csv")
  -skip-header
        skip header line in csv
  -url string
        url of sheet, e.g. "https://www.whooweswho.net/session#/sheets/1234/6789/expenses"
  -v    verbose output
  ```
