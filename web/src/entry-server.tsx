import { createHandler, StartServer } from "@solidjs/start/server";

export default createHandler(() => (
  <StartServer
    document={({ assets, children, scripts }) => (
      <html lang="en">
        <head>
          <meta charset="utf-8" />
          <meta name="viewport" content="width=device-width, initial-scale=1" />
          <title>AgentLoop</title>
          {assets}
        </head>
        <body class="bg-gray-950 text-gray-100 antialiased">
          <div id="app">{children}</div>
          {scripts}
        </body>
      </html>
    )}
  />
));
