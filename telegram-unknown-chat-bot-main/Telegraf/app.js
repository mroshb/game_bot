const { Telegraf, Scenes, session } = require("telegraf");
const dotenv = require("dotenv");
const {
  findLookingUser,
  createLookingUser,
  findLookingUserByGender,
  deleteLookingUser,
} = require("./utils/lookingUserUtil");
const {
  findActiveChat,
  createActiveChat,
  deleteActiveChat,
} = require("./utils/activeChatUtil");
const { findUser } = require("./utils/userUtil");
const { updateOptions, homeOptions } = require("./utils/updateOptions");
const { isUserRegistered } = require("./utils/userUtil");
const { isUserInWizard } = require("./utils/chatUtils");

dotenv.config();

const bot = new Telegraf(process.env.BOT_TOKEN);
const stage = new Scenes.Stage();

const startChatWizard = new Scenes.WizardScene(
  "startChat",
  async (ctx) => {
    if (!(await isUserRegistered(ctx.message.from.id))) {
      await ctx.reply("First, register!", updateOptions(["Register"]));
      return await ctx.scene.leave();
    }

    ctx.session.wizard = true;

    await ctx.reply(
      "Who would you like to chat with?",
      updateOptions(["Boy", "Girl", "Cancel"])
    );

    return ctx.wizard.steps[1](ctx);
  },
  async (ctx) => {
    if (ctx.message.text == "Cancel") {
      await ctx.reply("Operation cancelled", homeOptions());
      return ctx.scene.leave();
    }
    let requestedGender = ctx.message.text;
    if (requestedGender != "Boy" && requestedGender != "Girl") {
      await ctx.reply("Choose either Boy or Girl.");
      return ctx.wizard.selectStep(1);
    }

    await ctx.reply("Searching started", updateOptions(["Cancel"]));

    return ctx.wizard.steps[2](ctx);
  },
  async (ctx) => {
    let requestedGender = ctx.message.text;
    if (requestedGender == "Boy") {
      requestedGender = "male";
    } else {
      requestedGender = "female";
    }

    const user = await findUser(ctx.message.from.id);
    const result = await createLookingUser(
      user.telegramId,
      user.fullname,
      user.gender,
      requestedGender
    );

    return ctx.wizard.steps[3](ctx);
  },
  async (ctx) => {
    const checkUserActivity = async () => {
      const isUserInChat = await findActiveChat(ctx.message.from.id);
      if (isUserInChat) {
        clearInterval(intervalId);
        return await ctx.wizard.steps[4](ctx);
      }
      const user = await findUser(ctx.message.from.id);
      let requestedGender = await findLookingUser(ctx.message.from.id);
      requestedGender = requestedGender.requestedGender;
      const lookingUser = await findLookingUserByGender(
        requestedGender,
        ctx.message.from.id
      );
      if (
        !lookingUser ||
        lookingUser.telegramId == ctx.message.from.id ||
        lookingUser.requestedGender != user.gender
      ) {
        return false;
      } else {
        clearInterval(intervalId);
        await createActiveChat(ctx.message.from.id, lookingUser.telegramId);
        await ctx.wizard.steps[4](ctx);
        return true;
      }
    };

    const intervalId = setInterval(async () => {
      await checkUserActivity();
    }, 1000);

    ctx.scene.leave();
  },
  async (ctx) => {
    const activechat = await findActiveChat(ctx.message.from.id);
    await ctx.reply(
      "Start the conversation with greetings and respect",
      updateOptions(["End Chat"])
    );
    await deleteLookingUser(activechat.telegramId_1);
    await deleteLookingUser(activechat.telegramId_2);
    return ctx.wizard.steps[5](ctx);
  },
  async (ctx) => {
    bot.on("message", async (ctx) => {
      const activechat = await findActiveChat(ctx.message.from.id);
      if (activechat) {
        const targetId =
          ctx.message.from.id == activechat.telegramId_1
            ? activechat.telegramId_2
            : activechat.telegramId_1;
        ctx.telegram.sendCopy(
          targetId,
          ctx.message,
          updateOptions["End Chat"]
        );
      } else {
        delete ctx.session.wizard;
        return ctx.scene.leave();
      }
    });
  }
);

startChatWizard.leave(async (ctx) => {
  ctx.session.wizard = false;
});

//////////////////////////////////////////////////

// Add wizards to the stage
stage.register(require("./wizards/registrationWizard").registrationWizard);
stage.register(startChatWizard);

// Register the stage with the bot
bot.use(session());
bot.use(stage.middleware());

// Start the bot
bot.start(isUserInWizard, async (ctx) => {
  if (await isUserRegistered(ctx.message.from.id)) {
    ctx.reply("What can I do for you?", homeOptions());
  } else {
    ctx.reply(
      "Welcome to {Bot Name}! Click Register to get started",
      updateOptions(["Register"])
    );
  }
});

// Register the startChatWizard handler
bot.hears("Start searching for someone to chat", (ctx) =>
  ctx.scene.enter("startChat")
);

// Register the registrationWizard handler
bot.hears("Register", isUserInWizard, (ctx) => ctx.scene.enter("registration"));

bot.hears("Send a quick message to each other", async (ctx) => {
  const activechat = await findActiveChat(ctx.message.from.id);
  if (activechat) {
    await deleteLookingUser(activechat.telegramId_1);
    await deleteLookingUser(activechat.telegramId_2);
    await ctx.telegram.sendMessage(
      activechat.telegramId_1,
      "The game is over, go home",
      homeOptions()
    );
    await ctx.telegram.sendMessage(
      activechat.telegramId_2,
      "The game is over, go home",
      homeOptions()
    );
    await deleteActiveChat(ctx.message.from.id);
    return ctx.scene.leave(); // leave the scene to end the conversation
  }
});

bot.hears("Cancel", async (ctx) => {
  const lookingUser = await findLookingUser(ctx.message.from.id);
  if (lookingUser) {
    await ctx.reply("Operation cancelled", homeOptions());
    await deleteLookingUser(ctx.message.from.id);
    return ctx.scene.leave();
  }
});

// Start the bot
bot.launch();
