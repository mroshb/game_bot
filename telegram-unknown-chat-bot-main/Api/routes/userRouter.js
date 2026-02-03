const express = require("express");

const userController = require("../controllers/userController");

const router = express.Router();

// GET - /admin/users - shows all the sers
router.get("/", userController.getAllUsers);

// POST - /admin/users/create - creates a new user
router.post("/create", userController.createUser);

// GET - /admin/users/find/:id - finds a single user
router.get("/find/:id", userController.findUser);

// PUT - /admin/users/edit/:id - edits a user
router.put("/edit/:id", userController.editUser);

// DELETE - /admin/users/delete/:id - deletes a user
router.delete("/delete/:id", userController.deleteUser);

module.exports = router;
