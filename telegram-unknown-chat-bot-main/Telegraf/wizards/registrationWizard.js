const { Scenes } = require("telegraf");
const axios = require("axios");

const { updateOptions, homeOptions } = require("../utils/updateOptions");
const { isUserRegistered, createUser } = require("../utils/userUtil");

let gender, fullname, age, city;

// Create the registration wizard
const registrationWizard = new Scenes.WizardScene(
  "registration",
  async (ctx) => {
    ctx.session.wizard = true;
    if (ctx.message.text == "Cancel") {
      ctx.scene.leave();
      return ctx.reply(
        "Operation cancelled. Click /start to begin again",
        updateOptions(["Registration"])
      );
    }

    if (await isUserRegistered(ctx.message.from.id)) {
      await ctx.reply("You are already registered!");
      await ctx.reply("/start");
      return ctx.scene.leave();
    }

    ctx.reply(
      "Specify your gender",
      updateOptions(["Male", "Female", "Cancel"])
    );

    return ctx.wizard.next();
  },
  async (ctx) => {
    if (ctx.message.text == "Cancel") {
      ctx.scene.leave();
      return ctx.reply(
        "Operation cancelled. Click /start to begin again",
        updateOptions(["Registration"])
      );
    }

    gender = ctx.message.text;
    if (gender != "Male" && gender != "Female") {
      await ctx.reply("Choose either Male or Female.");
      return await ctx.wizard.selectStep(1);
    }

    if (gender == "Male") {
      gender = "male";
    } else {
      gender = "female";
    }

    ctx.reply(`Now, tell me your full name`, updateOptions(["Cancel"]));
    return ctx.wizard.next();
  },
  (ctx) => {
    if (ctx.message.text == "Cancel") {
      ctx.scene.leave();
      return ctx.reply(
        "Operation cancelled. Click /start to begin again",
        updateOptions(["Registration"])
      );
    }

    fullname = ctx.message.text;
    ctx.reply(
      `Alright, ${fullname}. Now, send me your age.`,
      updateOptions(["Cancel"])
    );
    return ctx.wizard.next();
  },
  async (ctx) => {
    if (ctx.message.text == "Cancel") {
      ctx.scene.leave();
      return ctx.reply(
        "Operation cancelled. Click /start to begin again",
        updateOptions(["Registration"])
      );
    }

    age = ctx.message.text;
    age = +age;
    if (isNaN(age)) {
      await ctx.reply("Only send a number!");
      return await ctx.wizard.selectStep(3);
    }
    ctx.reply(
      `Now that we know you are ${age} years old, tell me which city you are from.`,
      updateOptions(["Cancel"])
    );
    return ctx.wizard.next();
  },
  async (ctx) => {
    if (ctx.message.text == "Cancel") {
      ctx.scene.leave();
      return ctx.reply(
        "Operation cancelled. Click /start to begin again",
        updateOptions(["Registration"])
      );
    }

    city = ctx.message.text;

    const result = await createUser(
      fullname,
      gender,
      age,
      city,
      ctx.message.from.id
    );

    if (result) {
      ctx.reply(`Congratulations! You are now registered!`, homeOptions());
    } else {
      ctx.reply(
        "There was an issue. Please try again.",
        updateOptions(["Registration"])
      );
    }

    return ctx.scene.leave();
  }
);

registrationWizard.leave(async (ctx) => {
  ctx.session.wizard = false;
});

module.exports = {
  registrationWizard,
};
