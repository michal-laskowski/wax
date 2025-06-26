export class SomeClass { 
    private toReturn : any
    
    constructor(x) {
      this.toReturn = x ?? "- derfault -"
    }

    getValue(){
        return this.toReturn
    }

}


export const done = "a is done";