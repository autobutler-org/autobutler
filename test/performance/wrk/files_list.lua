counter = 0

request = function()
  counter = counter + 1
  local root = ""
  if counter % 3 == 0 then
    root = "perf"
  elseif counter % 5 == 0 then
    root = "perf/nested"
  end

  return wrk.format(
    "GET",
    "/api/v1/files?rootDir=" .. root
  )
end
