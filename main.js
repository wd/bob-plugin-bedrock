/**
 * Bob Translator Plugin for AWS Bedrock
 * Uses local bridge server for AWS authentication
 */

function translate(query, completion) {
  var serverUrl = $option.serverUrl || "http://localhost:18081";
  var model = $option.model || "anthropic.claude-3-haiku-20240307-v1:0";
  var prompt = $option.prompt || "";

  var requestBody = {
    text: query.text,
    from: query.detectFrom,
    to: query.detectTo,
    model: model,
    stream: true,
    prompt: prompt
  };

  var resultText = "";
  var streamError = null;

  $http.streamRequest({
    method: "POST",
    url: serverUrl + "/translate",
    header: {
      "Content-Type": "application/json"
    },
    body: requestBody,
    streamHandler: function(streamData) {
      // Parse SSE data chunks
      // Format: data: {"content": "..."}\n\n
      var text = streamData.text;
      if (!text) return;

      var lines = text.split("\n");
      for (var i = 0; i < lines.length; i++) {
        var line = lines[i].trim();
        if (line.indexOf("data: ") === 0) {
          var jsonStr = line.substring(6);
          if (jsonStr === "[DONE]") continue;

          try {
            var data = JSON.parse(jsonStr);
            if (data.error) {
              streamError = data.error;
            } else if (data.content) {
              resultText += data.content;
              query.onStream({
                result: {
                  toParagraphs: [resultText]
                }
              });
            }
          } catch (e) {
            // Skip invalid JSON chunks
            $log.error("Failed to parse SSE chunk: " + jsonStr);
          }
        }
      }
    },
    handler: function(response) {
      if (response.error) {
        var errorMessage = response.error.localizedDescription || "Network request failed";
        completion({
          error: {
            type: "network",
            message: errorMessage
          }
        });
        return;
      }

      // Check HTTP status code
      if (response.response && response.response.statusCode !== 200) {
        var statusCode = response.response.statusCode;
        var errorBody = "";

        try {
          if (response.data) {
            errorBody = JSON.stringify(response.data);
          }
        } catch (e) {
          errorBody = String(response.data || "");
        }

        completion({
          error: {
            type: "api",
            message: "Server returned status " + statusCode + ": " + errorBody
          }
        });
        return;
      }

      // Check for streaming error
      if (streamError) {
        completion({
          error: {
            type: "api",
            message: streamError
          }
        });
        return;
      }

      // Complete with final result
      if (resultText) {
        completion({
          result: {
            toParagraphs: [resultText]
          }
        });
      } else {
        // Try to parse non-streaming response
        try {
          var data = response.data;
          if (data && data.translation) {
            completion({
              result: {
                toParagraphs: [data.translation]
              }
            });
          } else {
            completion({
              error: {
                type: "api",
                message: "No translation received from server"
              }
            });
          }
        } catch (e) {
          completion({
            error: {
              type: "unknown",
              message: "Failed to parse response: " + e.message
            }
          });
        }
      }
    }
  });
}

function supportLanguages() {
  // Return all commonly supported language codes
  return [
    "auto",
    "zh-Hans",
    "zh-Hant",
    "en",
    "ja",
    "ko",
    "fr",
    "de",
    "es",
    "it",
    "ru",
    "pt",
    "nl",
    "pl",
    "ar",
    "th",
    "vi",
    "id",
    "ms",
    "tr",
    "cs",
    "uk",
    "sv",
    "da",
    "fi",
    "no",
    "hu",
    "el",
    "ro",
    "sk",
    "bg",
    "hr",
    "sl",
    "he",
    "hi",
    "bn"
  ];
}
