// Node Requirements
const path = require("path");

// Outer Requirements
const express = require("express");
const mongoose = require("mongoose");
const dotEnv = require("dotenv");

// Inner Requirements
const connectDB = require("./config/db");
const { errorHandler } = require("./middlewares/errors");
const { setHeaders } = require("./middlewares/headers");

// Load Config
dotEnv.config({ path: "./config/config.env" });

// ENVs
const PORT = process.env.PORT || 3000;
const NODE_ENV = process.env.NODE_ENV;

// Database Connection
connectDB();

// App
const app = express();

// BodyParser
app.use(express.urlencoded({ extended: false }));
app.use(express.json());
app.use(setHeaders);

// Static Folder
app.use(express.static(path.join(__dirname, "public")));

// Routes
app.use("/admin/users", require("./routes/userRouter"));
app.use("/admin/lookingusers", require("./routes/lookingUserRouter"));
app.use("/admin/activechats", require("./routes/activeChatRouter"));

// Error Controller
app.use(errorHandler);

// Port Settings
app.listen(PORT, () => {
  console.log("***********************");
  console.log(`Server started on ${PORT} && running on ${NODE_ENV} mode`);
});
