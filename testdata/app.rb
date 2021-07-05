#!/usr/bin/env ruby
#
# This is a mock for the Cambium login.
# It is deliberately slow.

require "sinatra"
require "sinatra/cookies"

get "/" do
  sleep 1
  redirect "/login", 303
end

get "/login" do
  erb :login
end

get "/cn-rtr/sso" do
  erb :sso
end

post "/login" do
  sleep 3
  cookies.set "sid",        value: "s:1234+y",     httponly: false
  cookies.set "XSRF-TOKEN", value: "asdfadfsadsf", httponly: false
  redirect "/app"
end

get "/app" do
  sleep 1
  erb :app
end
