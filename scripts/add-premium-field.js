// Script para agregar el campo 'premium' a todos los topics
// Ejecutar con: node add-premium-field.js [DB_URL]
// Ejemplo: node add-premium-field.js "mongodb+srv://user:pass@cluster.mongodb.net/opo"

const { MongoClient } = require('mongodb');
const fs = require('fs');
const path = require('path');

async function addPremiumField() {
    // Intentar obtener DB_URL de múltiples fuentes
    let uri = process.argv[2]; // Desde argumento de línea de comandos
    
    if (!uri) {
        // Intentar leer del archivo .env
        try {
            const envPath = path.join(__dirname, '..', '.env');
            const envContent = fs.readFileSync(envPath, 'utf8');
            const dbUrlMatch = envContent.match(/DB_URL=(.+)/);
            const mongoUrlMatch = envContent.match(/MONGO_URL=(.+)/);
            uri = (dbUrlMatch && dbUrlMatch[1]) || (mongoUrlMatch && mongoUrlMatch[1]);
        } catch (error) {
            // Archivo .env no encontrado o error leyéndolo
        }
    }
    
    if (!uri) {
        uri = process.env.DB_URL || process.env.MONGO_URL;
    }

    if (!uri) {
        console.error('❌ Error: No se encontró DB_URL');
        console.log('\n💡 Ejecuta el script de una de estas formas:');
        console.log('   1. node add-premium-field.js "mongodb+srv://user:pass@cluster.mongodb.net/opo"');
        console.log('   2. Configura DB_URL en el archivo .env');
        console.log('   3. Configura la variable de entorno DB_URL');
        process.exit(1);
    }
    
    const dbName = process.env.DB_NAME || 'opo';

    console.log('🔗 Conectando a MongoDB...');
    console.log(`📦 Base de datos: ${dbName}`);
    
    const client = new MongoClient(uri);

    try {
        await client.connect();
        console.log('✅ Conectado a MongoDB');

        const database = client.db(dbName);
        const collection = database.collection('topics_uuid_map');

        // Contar documentos sin el campo premium
        const countWithoutPremium = await collection.countDocuments({ 
            premium: { $exists: false } 
        });
        console.log(`\n📊 Topics sin campo 'premium': ${countWithoutPremium}`);

        if (countWithoutPremium === 0) {
            console.log('✅ Todos los topics ya tienen el campo premium definido');
            return;
        }

        // Actualizar todos los documentos que no tienen el campo premium
        const result = await collection.updateMany(
            { premium: { $exists: false } },
            { $set: { premium: false } }
        );

        console.log(`\n✅ Operación completada:`);
        console.log(`   - Documentos encontrados: ${result.matchedCount}`);
        console.log(`   - Documentos actualizados: ${result.modifiedCount}`);

        // Verificar el resultado
        const totalTopics = await collection.countDocuments({});
        const premiumTopics = await collection.countDocuments({ premium: true });
        const noPremiumTopics = await collection.countDocuments({ premium: false });

        console.log(`\n📊 Estadísticas finales:`);
        console.log(`   - Total de topics: ${totalTopics}`);
        console.log(`   - Topics premium: ${premiumTopics}`);
        console.log(`   - Topics no premium: ${noPremiumTopics}`);

        // Mostrar algunos ejemplos
        console.log(`\n🔍 Ejemplos de topics actualizados:`);
        const samples = await collection.find({ premium: false }).limit(3).toArray();
        samples.forEach((topic, index) => {
            console.log(`   ${index + 1}. ID: ${topic.id}, Title: ${topic.title}, Premium: ${topic.premium}`);
        });

    } catch (error) {
        console.error('❌ Error:', error);
        process.exit(1);
    } finally {
        await client.close();
        console.log('\n🔌 Conexión cerrada');
    }
}

addPremiumField();

