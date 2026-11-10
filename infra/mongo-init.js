const solar = db.getSiblingDB('solar_db');
solar.createUser({user:'solar',pwd:process.env.MONGO_SOLAR_PASSWORD,roles:[{role:'readWrite',db:'solar_db'}]});
