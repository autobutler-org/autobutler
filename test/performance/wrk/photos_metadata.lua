paths = {
  "perf/sample-1.jpg",
  "perf/sample-2.jpg",
  "perf/nested/sample-3.jpg"
}

counter = 0

request = function()
  counter = counter + 1
  local idx = (counter % #paths) + 1
  return wrk.format("GET", "/api/v1/photos/metadata?relPath=" .. paths[idx])
end
