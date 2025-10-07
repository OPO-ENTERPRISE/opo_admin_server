const { MongoClient } = require('mongodb');

async function checkUser() {
    const uri = "mongodb+srv://terro:Terro1975%24@cluster0.8s3fkqv.mongodb.net/opo?retryWrites=true&w=majority&tls=true";
    const client = new MongoClient(uri);

    try {
        await client.connect();
        console.log("✅ Conectado a MongoDB Atlas");

        const database = client.db('opo');
        const usersCollection = database.collection('user');

        console.log("\n🔍 Buscando usuario con email: superadmin@opo.com");
        
        const user = await usersCollection.findOne({ email: "superadmin@opo.com" });
        
        if (user) {
            console.log("✅ Usuario encontrado:");
            console.log(JSON.stringify(user, null, 2));
            
            console.log("\n🔍 Estructura de campos:");
            Object.keys(user).forEach(key => {
                console.log(`- ${key}: ${typeof user[key]} = ${user[key]}`);
            });
        } else {
            console.log("❌ Usuario no encontrado");
            
            // Buscar todos los usuarios para ver qué hay
            console.log("\n🔍 Todos los usuarios en la colección:");
            const allUsers = await usersCollection.find({}).toArray();
            allUsers.forEach((u, index) => {
                console.log(`Usuario ${index + 1}:`);
                console.log(JSON.stringify(u, null, 2));
                console.log("---");
            });
        }

    } catch (e) {
        console.error("❌ Error:", e);
    } finally {
        await client.close();
        console.log("\n🔌 Conexión cerrada");
    }
}

checkUser();
