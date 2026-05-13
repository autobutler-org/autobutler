counter = 0

request = function()
  counter = counter + 1
  local offset = ((counter - 1) % 10) * 10
  return wrk.format(
    "GET",
    "/api/v1/photos?limit=50&offset=" .. offset
  )
end
