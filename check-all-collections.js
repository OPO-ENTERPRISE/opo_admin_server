const { MongoClient } = require('mongodb');

async function checkAllCollections() {
    const uri = "mongodb+srv://terro:Terro1975%24@cluster0.8s3fkqv.mongodb.net/opo?retryWrites=true&w=majority&tls=true";
    const client = new MongoClient(uri);

    try {
        await client.connect();
        console.log("✅ Conectado a MongoDB Atlas");

        const database = client.db('opo');
        
        // Listar todas las colecciones
        const collections = await database.listCollections().toArray();
        console.log("\n📋 Colecciones disponibles:");
        collections.forEach(col => {
            console.log(`- ${col.name}`);
        });

        // Buscar en cada colección que pueda contener usuarios
        const possibleCollections = ['user', 'users', 'admin', 'admins'];
        
        for (const collectionName of possibleCollections) {
            try {
                const collection = database.collection(collectionName);
                const count = await collection.countDocuments();
                console.log(`\n🔍 Colección '${collectionName}': ${count} documentos`);
                
                if (count > 0) {
                    const docs = await collection.find({}).limit(5).toArray();
                    docs.forEach((doc, index) => {
                        console.log(`Documento ${index + 1}:`);
                        console.log(JSON.stringify(doc, null, 2));
                        console.log("---");
                    });
                }
            } catch (e) {
                console.log(`❌ Error accediendo a colección '${collectionName}': ${e.message}`);
            }
        }

        // Buscar específicamente por email en todas las colecciones
        console.log("\n🔍 Buscando por email 'superadmin@opo.com' en todas las colecciones:");
        for (const collectionName of possibleCollections) {
            try {
                const collection = database.collection(collectionName);
                const user = await collection.findOne({ email: "superadmin@opo.com" });
                if (user) {
                    console.log(`✅ Encontrado en colección '${collectionName}':`);
                    console.log(JSON.stringify(user, null, 2));
                }
            } catch (e) {
                // Ignorar errores de colección no existente
            }
        }

    } catch (e) {
        console.error("❌ Error:", e);
    } finally {
        await client.close();
        console.log("\n🔌 Conexión cerrada");
    }
}

checkAllCollections();
