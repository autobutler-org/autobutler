paths = {
  "/api/v1/thumbnails/perf/sample-1.jpg",
  "/api/v1/thumbnails/perf/sample-2.jpg",
  "/api/v1/thumbnails/perf/nested/sample-3.jpg"
}

counter = 0

request = function()
  counter = counter + 1
  local idx = (counter % #paths) + 1
  return wrk.format("GET", paths[idx])
end
