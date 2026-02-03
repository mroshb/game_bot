const express = require("express");

const lookingUserController = require("../controllers/lookigUserController");

const router = express.Router();

// GET - /admin/lookingusers/find/:id - finds a single looking user
router.get("/find/:id", lookingUserController.findLookingUser);

// POST - /admin/lookingusers/create - creates a new looking user
router.post("/create", lookingUserController.createLookigUser);

// DELETE - /admin/lookingusers/delete/:id - deletes a looking user
router.delete("/delete/:id", lookingUserController.deleteLookingUser);

// GET - /admin/lookingusers/findbygender - finds a single looking user
router.post("/findbygender", lookingUserController.findLookingUserByGender);

module.exports = router;
