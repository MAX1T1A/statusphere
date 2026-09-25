# work-runtime 2.7 (via Glance) and ML Kit ship class-only keep rules, which R8 full mode
# no longer reads as keeping the no-arg constructor they instantiate reflectively.
-keep class * extends androidx.room.RoomDatabase { <init>(); }
-keep class * implements com.google.firebase.components.ComponentRegistrar { <init>(); }
